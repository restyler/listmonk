<template>
  <section class="dynamic-segments">
    <header class="columns page-header">
      <div class="column is-10">
        <h1 class="title is-4">
          Dynamic Segments
          <span v-if="!isNaN(segments.total)">
            (<span data-cy="count">{{ segments.total }}</span>)
          </span>
        </h1>
      </div>
      <div class="column has-text-right">
        <b-field expanded>
          <b-button expanded type="is-primary" icon-left="plus" @click="showNewForm" data-cy="btn-new" class="btn-new">
            {{ $t('globals.buttons.new') }}
          </b-button>
        </b-field>
      </div>
    </header>

    <section class="segments-controls">
      <div class="columns">
        <div class="column is-8">
          <form @submit.prevent="onSubmit">
            <b-field addons>
              <b-input @input="onSearchInput" v-model="searchInput" expanded
                placeholder="Search segments..." icon="magnify" ref="search"
                data-cy="search" />
              <p class="controls">
                <b-button native-type="submit" type="is-primary" icon-left="magnify"
                  data-cy="btn-search" />
              </p>
            </b-field>
          </form>
        </div>
      </div>
    </section>

    <br />

        <b-table :data="segments.results ?? []" :loading="loading.segments" paginated backend-pagination
      pagination-position="both" @page-change="onPageChange" :current-page="queryParams.page"
      :per-page="segments.perPage" :total="segments.total" hoverable>
<b-table-column v-slot="props" field="name" label="Name" sortable>
        <strong>{{ props.row.name }}</strong>
        <br>
        <span class="is-size-7 has-text-grey">{{ props.row.description }}</span>
      </b-table-column>

      <b-table-column v-slot="props" field="list" label="Target List">
        <b-tag type="is-info">{{ props.row.list.name }}</b-tag>
      </b-table-column>

      <b-table-column v-slot="props" field="sql_snippet" label="SQL Snippet">
        <b-tag v-if="props.row.sqlSnippet" type="is-light">
          {{ props.row.sqlSnippet.name }}
        </b-tag>
        <span v-else class="has-text-grey">Custom Query</span>
      </b-table-column>

      <b-table-column v-slot="props" field="enabled" label="Status" centered>
        <b-tag :type="props.row.enabled ? 'is-success' : 'is-warning'">
          {{ props.row.enabled ? 'Enabled' : 'Disabled' }}
        </b-tag>
      </b-table-column>

      <b-table-column v-slot="props" field="created_at" label="Created" sortable>
        {{ $utils.niceDate(props.row.createdAt) }}
      </b-table-column>

      <b-table-column v-slot="props" cell-class="actions" align="right">
        <div>
          <a href="#" @click.prevent="showEditForm(props.row)" data-cy="btn-edit"
            :aria-label="$t('globals.buttons.edit')">
            <b-tooltip label="Edit" type="is-dark">
              <b-icon icon="pencil-outline" size="is-small" />
            </b-tooltip>
          </a>
          <a href="#" @click.prevent="deleteSegment(props.row)" data-cy="btn-delete"
            :aria-label="$t('globals.buttons.delete')">
            <b-tooltip label="Delete" type="is-dark">
              <b-icon icon="trash-can-outline" size="is-small" />
            </b-tooltip>
          </a>
        </div>
      </b-table-column>

      <template #empty v-if="!loading.segments">
        <empty-placeholder />
      </template>
    </b-table>

    <!-- Add / edit form modal -->
    <b-modal scroll="keep" :aria-modal="true" :active.sync="isFormVisible" :width="850" @close="onFormClose">
      <div class="modal-card" style="width: auto">
        <header class="modal-card-head">
          <p class="modal-card-title">
            {{ isEditing ? 'Edit Dynamic Segment' : 'New Dynamic Segment' }}
          </p>
        </header>
        <section class="modal-card-body">
          <form @submit.prevent="onSubmitForm">
            <b-field label="Name" :type="errors.name ? 'is-danger' : ''" :message="errors.name">
              <b-input v-model="form.name" :maxlength="200" required />
            </b-field>

            <b-field label="Description">
              <b-input v-model="form.description" type="textarea" :maxlength="500" />
            </b-field>

            <b-field label="Target List" :type="errors.listId ? 'is-danger' : ''" :message="errors.listId">
              <b-select v-model="form.listId" expanded required>
                <option value="">Select a list</option>
                <option v-for="list in lists" :key="list.id" :value="list.id">
                  {{ list.name }}
                </option>
              </b-select>
            </b-field>

            <b-field label="SQL Source">
              <b-radio-button v-model="form.sqlSource" native-value="snippet" type="is-light">
                Use SQL Snippet
              </b-radio-button>
              <b-radio-button v-model="form.sqlSource" native-value="custom" type="is-light">
                Custom Query
              </b-radio-button>
            </b-field>

            <b-field v-if="form.sqlSource === 'snippet'" label="SQL Snippet"
              :type="errors.sqlSnippetId ? 'is-danger' : ''" :message="errors.sqlSnippetId">
              <b-select v-model="form.sqlSnippetId" expanded>
                <option value="">Select a snippet</option>
                <option v-for="snippet in sqlSnippets" :key="snippet.id" :value="snippet.id">
                  {{ snippet.name }}
                </option>
              </b-select>
            </b-field>

            <b-field v-if="form.sqlSource === 'custom'" label="Custom SQL WHERE Condition"
              :type="errors.query ? 'is-danger' : ''" :message="errors.query">
              <b-input v-model="form.query" type="textarea" :rows="6" required />
              <p class="help">
                Enter only the WHERE condition part. Examples: <code>status = 'confirmed'</code>, <code>(subscribers.attribs->>'age')::INT > 39</code>
              </p>
            </b-field>

            <b-field>
              <b-checkbox v-model="form.enabled">
                Enable this segment
              </b-checkbox>
            </b-field>
          </form>
        </section>
        <footer class="modal-card-foot">
          <b-button type="is-primary" :loading="loading.form" @click="onSubmitForm">
            {{ isEditing ? 'Update' : 'Create' }}
          </b-button>
          <b-button @click="hideForm">Cancel</b-button>
        </footer>
      </div>
    </b-modal>
  </section>
</template>

<script>
import Vue from 'vue';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';

export default Vue.extend({
  components: {
    EmptyPlaceholder,
  },

  data() {
    return {
      segments: {
        results: [],
        total: 0,
        perPage: 20,
      },
      lists: [],
      sqlSnippets: [],
      searchInput: '',
      queryParams: {
        page: 1,
        query: '',
      },
      isFormVisible: false,
      isEditing: false,
      form: this.getEmptyForm(),
      errors: {},
      loading: {
        segments: true,
        form: false,
      },
    };
  },

  methods: {
    getEmptyForm() {
      return {
        name: '',
        description: '',
        listId: '',
        sqlSnippetId: '',
        query: '',
        sqlSource: 'snippet',
        enabled: true,
      };
    },

    onSearchInput() {
      this.queryParams.query = this.searchInput;
      this.queryParams.page = 1;
      this.querySegments();
    },

    onSubmit() {
      this.querySegments();
    },

    onPageChange(page) {
      this.queryParams.page = page;
      this.querySegments();
    },

    async querySegments() {
      console.log('=== querySegments called in DynamicSegments ===');
      this.loading.segments = true;
      try {
        console.log('Attempting to load dynamic segments...');
        console.log('$api available:', !!this.$api);
        console.log('getDynamicSegments method available:', !!this.$api?.getDynamicSegments);

        if (!this.$api || !this.$api.getDynamicSegments) {
          console.error('API or getDynamicSegments method not available');
          this.segments = { results: [], total: 0, perPage: 20 };
          return;
        }

        const params = {
          page: this.queryParams.page,
          per_page: this.segments.perPage,
        };
        if (this.queryParams.query) {
          params.query = this.queryParams.query;
        }

        console.log('API params:', params);
        const response = await this.$api.getDynamicSegments(params);
        console.log('Dynamic segments response:', response);

        if (response) {
          this.segments = response;
          console.log('Dynamic segments loaded successfully:', this.segments.results?.length || 0, 'segments');
        } else {
          console.warn('Unexpected response structure:', response);
          this.segments = { results: [], total: 0, perPage: 20 };
        }
      } catch (e) {
        console.error('Error loading dynamic segments:', e);
        console.error('Error details:', {
          message: e.message,
          response: e.response,
          stack: e.stack,
        });
        this.segments = { results: [], total: 0, perPage: 20 };
        this.$utils.toast(e.message || 'Error loading segments', 'is-danger');
      }
      this.loading.segments = false;
    },

    async loadLists() {
      console.log('=== loadLists called in DynamicSegments ===');
      try {
        console.log('Attempting to load lists...');
        console.log('$api available:', !!this.$api);
        console.log('getLists method available:', !!this.$api?.getLists);

        if (!this.$api || !this.$api.getLists) {
          console.error('API or getLists method not available');
          this.lists = [];
          return;
        }

        const response = await this.$api.getLists({ per_page: 'all' });
        console.log('Lists response:', response);

        if (response && response.results) {
          this.lists = response.results;
          console.log('Lists loaded successfully:', this.lists.length, 'lists');
        } else {
          console.warn('Unexpected response structure:', response);
          this.lists = [];
        }
      } catch (e) {
        console.error('Error loading lists:', e);
        console.error('Error details:', {
          message: e.message,
          response: e.response,
          stack: e.stack,
        });
        this.lists = [];
        this.$utils.toast(e.message || 'Error loading lists', 'is-danger');
      }
    },

    async loadSqlSnippets() {
      console.log('=== loadSqlSnippets called in DynamicSegments ===');
      try {
        console.log('Attempting to load SQL snippets...');
        console.log('$api available:', !!this.$api);
        console.log('getSQLSnippets method available:', !!this.$api?.getSQLSnippets);

        if (!this.$api || !this.$api.getSQLSnippets) {
          console.error('API or getSQLSnippets method not available');
          this.sqlSnippets = [];
          return;
        }

        const response = await this.$api.getSQLSnippets({ per_page: 'all' });
        console.log('SQL snippets response:', response);

        if (response && response.results) {
          this.sqlSnippets = response.results;
          console.log('SQL snippets loaded successfully:', this.sqlSnippets.length, 'snippets');
        } else {
          console.warn('Unexpected response structure:', response);
          this.sqlSnippets = [];
        }
      } catch (e) {
        console.error('Error loading SQL snippets:', e);
        console.error('Error details:', {
          message: e.message,
          response: e.response,
          stack: e.stack,
        });
        this.sqlSnippets = [];
        this.$utils.toast(e.message || 'Error loading SQL snippets', 'is-danger');
      }
    },

    showNewForm() {
      this.isEditing = false;
      this.form = this.getEmptyForm();
      this.errors = {};
      this.isFormVisible = true;
    },

    showEditForm(segment) {
      console.log('=== showEditForm called in DynamicSegments ===');
      console.log('Segment data:', segment);

      this.isEditing = true;
      this.form = {
        id: segment.id,
        name: segment.name,
        description: segment.description,
        listId: segment.listId || segment.list_id || (segment.list ? segment.list.id : ''),
        sqlSnippetId: segment.snippetId || segment.snippet_id || (segment.sqlSnippet ? segment.sqlSnippet.id : ''),
        query: segment.query || '',
        sqlSource: (segment.snippetId || segment.snippet_id || segment.sqlSnippet) ? 'snippet' : 'custom',
        enabled: this.getSegmentEnabledValue(segment),
      };

      console.log('Form data after mapping:', this.form);
      this.errors = {};
      this.isFormVisible = true;
    },

    getSegmentEnabledValue(segment) {
      if (segment.isActive !== undefined) {
        return segment.isActive;
      }
      if (segment.is_active !== undefined) {
        return segment.is_active;
      }
      return true;
    },

    hideForm() {
      this.isFormVisible = false;
    },

    onFormClose() {
      this.hideForm();
    },

    async onSubmitForm() {
      console.log('=== onSubmitForm called in DynamicSegments ===');
      this.loading.form = true;
      this.errors = {};

      const data = { ...this.form };
      if (data.sqlSource === 'snippet') {
        data.query = '';
      } else {
        data.sqlSnippetId = null;
      }
      delete data.sqlSource;

      // Map frontend field names to backend expected field names
      data.list_id = data.listId;
      data.snippet_id = data.sqlSnippetId;
      data.is_active = data.enabled;
      delete data.listId;
      delete data.sqlSnippetId;
      delete data.enabled;

      console.log('Form data being submitted:', data);
      console.log('isEditing:', this.isEditing);

      try {
        console.log('$api available:', !!this.$api);

        if (this.isEditing) {
          console.log('updateDynamicSegment method available:', !!this.$api?.updateDynamicSegment);
          if (!this.$api || !this.$api.updateDynamicSegment) {
            console.error('API or updateDynamicSegment method not available');
            this.$utils.toast('API not available for updating segment', 'is-danger');
            return;
          }

          data.id = this.form.id;
          await this.$api.updateDynamicSegment(data);
          this.$utils.toast('Segment updated');
        } else {
          console.log('createDynamicSegment method available:', !!this.$api?.createDynamicSegment);
          if (!this.$api || !this.$api.createDynamicSegment) {
            console.error('API or createDynamicSegment method not available');
            this.$utils.toast('API not available for creating segment', 'is-danger');
            return;
          }

          await this.$api.createDynamicSegment(data);
          this.$utils.toast('Segment created');
        }
        this.hideForm();
        this.querySegments();
      } catch (e) {
        console.error('Error saving dynamic segment:', e);
        console.error('Error details:', {
          message: e.message,
          response: e.response,
          stack: e.stack,
        });
        if (e.response && e.response.data && e.response.data.data) {
          this.errors = e.response.data.data;
        } else {
          this.$utils.toast(e.message || 'Error saving segment', 'is-danger');
        }
      }
      this.loading.form = false;
    },

    async deleteSegment(segment) {
      this.$buefy.dialog.confirm({
        message: `Delete the segment <strong>${segment.name}</strong>? This cannot be undone.`,
        confirmText: 'Delete',
        type: 'is-danger',
        onConfirm: async () => {
          console.log('=== deleteSegment called in DynamicSegments ===');
          console.log('Segment to delete:', segment);

          try {
            console.log('$api available:', !!this.$api);
            console.log('deleteDynamicSegment method available:', !!this.$api?.deleteDynamicSegment);

            if (!this.$api || !this.$api.deleteDynamicSegment) {
              console.error('API or deleteDynamicSegment method not available');
              this.$utils.toast('API not available for deleting segment', 'is-danger');
              return;
            }

            await this.$api.deleteDynamicSegment(segment.id);
            this.$utils.toast('Segment deleted');
            this.querySegments();
          } catch (e) {
            console.error('Error deleting dynamic segment:', e);
            console.error('Error details:', {
              message: e.message,
              response: e.response,
              stack: e.stack,
            });
            this.$utils.toast(e.message || 'Error deleting segment', 'is-danger');
          }
        },
      });
    },
  },

  async mounted() {
    await Promise.all([
      this.querySegments(),
      this.loadLists(),
      this.loadSqlSnippets(),
    ]);
  },
});
</script>
