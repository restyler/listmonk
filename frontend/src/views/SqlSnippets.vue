<template>
  <section class="sql-snippets">
    <header class="columns page-header">
      <div class="column is-10">
        <h1 class="title is-4">
          SQL Snippets
          <span v-if="!isNaN(snippets.total)">
            (<span data-cy="count">{{ snippets.total }}</span>)
          </span>
        </h1>
        <p class="subtitle">
          Create reusable SQL queries for filtering subscribers
        </p>
      </div>
      <div class="column has-text-right">
        <b-field expanded>
          <b-button expanded type="is-primary" icon-left="plus" @click="showNewForm" data-cy="btn-new" class="btn-new">
            {{ $t('globals.buttons.new') }}
          </b-button>
        </b-field>
      </div>
    </header>

    <section class="snippets-controls">
      <div class="columns">
        <div class="column is-8">
          <form @submit.prevent="onSubmit">
            <b-field addons>
              <b-input @input="onSearchInput" v-model="searchInput" expanded
                placeholder="Search snippets..." icon="magnify" ref="search"
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

    <b-table :data="snippets.results ?? []" :loading="loading.snippets" paginated backend-pagination
      pagination-position="both" @page-change="onPageChange" :current-page="queryParams.page"
      :per-page="snippets.perPage" :total="snippets.total" hoverable>
<b-table-column v-slot="props" field="name" label="Name" sortable>
        <strong>{{ props.row.name }}</strong>
        <br>
        <span class="is-size-7 has-text-grey">{{ props.row.description }}</span>
      </b-table-column>

      <b-table-column v-slot="props" field="query" label="Query">
        <code class="is-size-7">{{ truncateQuery(props.row.querySql || props.row.query_sql || props.row.querySQL || props.row.query) }}</code>
        <br>
        <a href="#" @click.prevent="showQueryModal(props.row)" class="is-size-7">
          <b-icon icon="eye" size="is-small" />
          View full query
        </a>
      </b-table-column>

      <b-table-column v-slot="props" field="created_at" label="Created" sortable>
        {{ $utils.niceDate(props.row.createdAt) }}
      </b-table-column>

      <b-table-column v-slot="props" field="updated_at" label="Updated" sortable>
        {{ $utils.niceDate(props.row.updatedAt) }}
      </b-table-column>

      <b-table-column v-slot="props" cell-class="actions" align="right">
        <div>
          <a href="#" @click.prevent="testQuery(props.row)" data-cy="btn-test"
            aria-label="Test query">
            <b-tooltip label="Test query" type="is-dark">
              <b-icon icon="play-circle-outline" size="is-small" />
            </b-tooltip>
          </a>
          <a href="#" @click.prevent="showEditForm(props.row)" data-cy="btn-edit"
            :aria-label="$t('globals.buttons.edit')">
            <b-tooltip label="Edit" type="is-dark">
              <b-icon icon="pencil-outline" size="is-small" />
            </b-tooltip>
          </a>
          <a href="#" @click.prevent="deleteSnippet(props.row)" data-cy="btn-delete"
            :aria-label="$t('globals.buttons.delete')">
            <b-tooltip label="Delete" type="is-dark">
              <b-icon icon="trash-can-outline" size="is-small" />
            </b-tooltip>
          </a>
        </div>
      </b-table-column>

      <template #empty v-if="!loading.snippets">
        <empty-placeholder />
      </template>
    </b-table>

    <!-- Add / edit form modal -->
    <b-modal scroll="keep" :aria-modal="true" :active.sync="isFormVisible" :width="850" @close="onFormClose" @opened="onModalOpened">
      <div class="modal-card" style="width: auto">
        <header class="modal-card-head">
          <p class="modal-card-title">
            {{ isEditing ? 'Edit SQL Snippet' : 'New SQL Snippet' }}
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

            <b-field label="SQL WHERE Condition"
              message="Enter only the WHERE condition part. Examples: subscribers.status = 'enabled', (subscribers.attribs->'age')::INT > 39"
              :type="errors.query ? 'is-danger' : ''">
              <b-input v-model="form.query" type="textarea" rows="5"
                placeholder="subscribers.status = 'enabled'"
                :class="errors.query ? 'is-danger' : ''"
                style="width: 100%;"
                @input="onQueryInput" />
            </b-field>

            <div class="buttons">
              <b-button type="is-info" :loading="loading.test" @click="testCurrentQuery">
                Test Query
              </b-button>
            </div>

            <div v-if="testResult" class="notification" :class="testResult.success ? 'is-success' : 'is-danger'">
              <div v-if="testResult.success">
                <strong>Query validation successful!</strong>
                <br>
                {{ testResult.message }}
              </div>
              <div v-else>
                <strong>Query validation failed:</strong>
                <br>
                {{ testResult.error }}
              </div>
            </div>
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

    <!-- Query view modal -->
    <b-modal scroll="keep" :aria-modal="true" :active.sync="isQueryModalVisible" :width="700">
      <div class="modal-card" style="width: auto">
        <header class="modal-card-head">
          <p class="modal-card-title">{{ selectedSnippet?.name }}</p>
        </header>
        <section class="modal-card-body">
          <b-field label="Description" v-if="selectedSnippet?.description">
            <p>{{ selectedSnippet.description }}</p>
          </b-field>
          <b-field label="SQL Query">
            <pre class="code-block">{{ selectedSnippet?.querySql || selectedSnippet?.query_sql || selectedSnippet?.querySQL || selectedSnippet?.query }}</pre>
          </b-field>
        </section>
        <footer class="modal-card-foot">
          <b-button @click="isQueryModalVisible = false">Close</b-button>
        </footer>
      </div>
    </b-modal>
  </section>
</template>

<script>
import Vue from 'vue';
import {
  validateSQLSnippet, createSQLSnippet, updateSQLSnippet, getSQLSnippets, deleteSQLSnippet,
} from '../api';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';

export default Vue.extend({
  components: {
    EmptyPlaceholder,
  },

  data() {
    return {
      snippets: {
        results: [],
        total: 0,
        perPage: 20,
      },
      searchInput: '',
      queryParams: {
        page: 1,
        query: '',
      },
      isFormVisible: false,
      isQueryModalVisible: false,
      isEditing: false,
      selectedSnippet: null,
      form: this.getEmptyForm(),
      errors: {},
      testResult: null,
      loading: {
        snippets: true,
        form: false,
        test: false,
      },
    };
  },

  methods: {
    getEmptyForm() {
      return {
        name: '',
        description: '',
        query: '',
      };
    },

    truncateQuery(query) {
      if (!query) return '';
      return query.length > 100 ? `${query.substring(0, 100)}...` : query;
    },

    onSearchInput() {
      this.queryParams.query = this.searchInput;
      this.queryParams.page = 1;
      this.querySnippets();
    },

    onSubmit() {
      this.querySnippets();
    },

    onPageChange(page) {
      this.queryParams.page = page;
      this.querySnippets();
    },

    async querySnippets() {
      this.loading.snippets = true;
      try {
        const params = {
          page: this.queryParams.page,
          per_page: this.snippets.perPage,
        };
        if (this.queryParams.query) {
          params.query = this.queryParams.query;
        }

        const response = await getSQLSnippets(params);
        this.snippets = response;
      } catch (e) {
        console.error('Error loading SQL snippets:', e);
        this.$utils.toast(e.message || 'Error loading snippets', 'is-danger');
      }
      this.loading.snippets = false;
    },

    showNewForm() {
      this.isEditing = false;
      this.form = this.getEmptyForm();
      this.errors = {};
      this.testResult = null;
      this.isFormVisible = true;
    },

    showEditForm(snippet) {
      console.log('=== showEditForm called ===');
      console.log('Snippet object:', snippet);
      console.log('snippet.querySql:', snippet.querySql);
      console.log('snippet.query_sql:', snippet.query_sql);
      console.log('snippet.querySQL:', snippet.querySQL);
      console.log('snippet.query:', snippet.query);
      this.isEditing = true;
      this.form = {
        id: snippet.id,
        name: snippet.name,
        description: snippet.description,
        query: snippet.querySql || snippet.query_sql || snippet.querySQL || snippet.query,
      };

      console.log('Form after assignment:', this.form);
      console.log('Form query value:', this.form.query);

      this.errors = {};
      this.testResult = null;
      this.isFormVisible = true;

      // Add a timeout to check if the form data is still there after Vue reactivity
      setTimeout(() => {
        console.log('Form data after timeout:', this.form);
        console.log('Form query after timeout:', this.form.query);
      }, 100);
    },

    showQueryModal(snippet) {
      this.selectedSnippet = snippet;
      this.isQueryModalVisible = true;
    },

    hideForm() {
      this.isFormVisible = false;
    },

    onModalOpened() {
      console.log('=== Modal opened ===');
      console.log('isEditing:', this.isEditing);
      console.log('Form data when modal opened:', this.form);
      console.log('Form query when modal opened:', this.form.query);
    },

    onFormClose() {
      this.hideForm();
    },

    async testQuery(snippet) {
      await this.testSqlQuery(snippet.querySql || snippet.query_sql || snippet.querySQL || snippet.query);
    },

    async testCurrentQuery() {
      await this.testSqlQuery(this.form.query);
    },

    async testSqlQuery(query) {
      if (!query.trim()) {
        this.$utils.toast('Please enter a query to test', 'is-warning');
        return;
      }

      this.loading.test = true;
      this.testResult = null;

      try {
        const response = await validateSQLSnippet({ query_sql: query });

        this.testResult = {
          success: true,
          message: response.message || 'Query is valid',
        };
        this.$utils.toast('Query validation successful!', 'is-success');
      } catch (e) {
        console.error('Error validating SQL query:', e);
        this.testResult = {
          success: false,
          error: e.response?.data?.message || e.message || 'Query validation failed',
        };
        this.$utils.toast(this.testResult.error, 'is-danger');
      } finally {
        this.loading.test = false;
      }
    },

    onQueryInput(value) {
      console.log('Query input changed:', value);
      console.log('Form.query is now:', this.form.query);
    },

    async onSubmitForm() {
      console.log('=== onSubmitForm called ===');
      console.log('Form data at submit:', this.form);
      this.loading.form = true;
      this.errors = {};

      try {
        // Map form field names to match API expectations
        const data = {
          name: this.form.name,
          description: this.form.description,
          query_sql: this.form.query,
        };

        console.log('Data being sent to API:', data);

        if (this.isEditing) {
          data.id = this.form.id;
          await updateSQLSnippet(data);
          this.$utils.toast('Snippet updated');
        } else {
          await createSQLSnippet(data);
          this.$utils.toast('Snippet created');
        }
        this.hideForm();
        this.querySnippets();
      } catch (e) {
        console.error('Error saving SQL snippet:', e);
        if (e.response && e.response.data && e.response.data.data) {
          this.errors = e.response.data.data;
        } else {
          this.$utils.toast(e.message || 'Error saving snippet', 'is-danger');
        }
      }
      this.loading.form = false;
    },

    async deleteSnippet(snippet) {
      this.$buefy.dialog.confirm({
        message: `Delete the snippet <strong>${snippet.name}</strong>? This cannot be undone.`,
        confirmText: 'Delete',
        type: 'is-danger',
        onConfirm: async () => {
          try {
            await deleteSQLSnippet(snippet.id);
            this.$utils.toast('Snippet deleted');
            this.querySnippets();
          } catch (e) {
            console.error('Error deleting SQL snippet:', e);
            this.$utils.toast(e.message || 'Error deleting snippet', 'is-danger');
          }
        },
      });
    },
  },

  mounted() {
    this.querySnippets();
  },
});
</script>

<style lang="scss" scoped>
.code-block {
  background: #f5f5f5;
  padding: 1rem;
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 14px;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
